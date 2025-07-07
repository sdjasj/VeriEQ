`timescale 1ns/1ps
`define stop   $stop
`define checkd(gotv, expv) \
  do if ((gotv) !== (expv)) begin \
       $write("%%Error: %s:%0d:  got=%0d exp=%0d\n", `__FILE__, `__LINE__, (gotv), (expv)); \
       `stop; \
     end while(0);
module top (out33);

output wire [6:0] out33;

    assign out33 = (6'o66 <<< 32'hFFFF_FFFF);

    initial begin
      #10;
      `checkd(out33, '0);
      $write("*-* All Finished *-*\n");
      $finish; 
    end

endmodule
