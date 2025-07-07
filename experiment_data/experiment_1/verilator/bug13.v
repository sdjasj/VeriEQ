`timescale 1ns/1ps

module top (out0);
  output wire [32'hFFFF_FFFF:1] out0;

  assign out0 = 1;

  initial begin
    #10;
    $display("out0: %b", out0);
    $finish; 
  end

endmodule