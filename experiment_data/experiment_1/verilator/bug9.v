`timescale 1ns/1ps
`define stop   $stop
`define checkd(gotv, expv) \
  do if ((gotv) !== (expv)) begin \
       $write("%%Error: %s:%0d:  got=%0d exp=%0d\n", `__FILE__, `__LINE__, (gotv), (expv)); \
       `stop; \
     end while(0);

module top ();

    reg  in4;
    wire signed wire_2;
    wire out88;

    assign wire_2 = in4 ? 2'b10 : 0;
    assign out88 = (-(wire_2) <= wire_2 ? 1 : 0);

    initial begin
        in4 = 1'b0;
        #5;
        `checkd(out88, '1);
        $write("*-* All Finished *-*\n");
        $finish;
    end

endmodule