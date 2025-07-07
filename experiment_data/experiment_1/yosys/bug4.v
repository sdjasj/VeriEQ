module a (
    output b
);
    reg [2:1] c;
    
    always begin
        case (0)
            default: begin
                b = c;
                c = ^c;
            end
        endcase
    end
endmodule